package main

import (
	"bytes"
	"io"
	"net"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// lockedBuffer — потокобезопасный writer для тестов run(). run() возвращается,
// как только завершается одна из горутин пересылки, поэтому вторая горутина
// может всё ещё быть заблокирована в io.Copy и обратиться к writer позже.
type lockedBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (b *lockedBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Write(p)
}

func (b *lockedBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.String()
}

func TestTelnetClient(t *testing.T) {
	t.Run("basic", func(t *testing.T) {
		l, err := net.Listen("tcp", "127.0.0.1:")
		require.NoError(t, err)
		defer func() { require.NoError(t, l.Close()) }()

		var wg sync.WaitGroup
		wg.Add(2)

		go func() {
			defer wg.Done()

			in := &bytes.Buffer{}
			out := &bytes.Buffer{}

			// assert (в отличие от require) безопасно вызывать из горутины, не
			// выполняющей тест: require вызвал бы t.FailNow из чужой горутины,
			// скрыв ошибку или подвесив wg.Wait().
			timeout, err := time.ParseDuration("10s")
			if !assert.NoError(t, err) {
				return
			}

			client := NewTelnetClient(l.Addr().String(), timeout, io.NopCloser(in), out)
			if !assert.NoError(t, client.Connect()) {
				return
			}
			defer func() { assert.NoError(t, client.Close()) }()

			in.WriteString("hello\n")
			if !assert.NoError(t, client.Send()) {
				return
			}

			if !assert.NoError(t, client.Receive()) {
				return
			}
			assert.Equal(t, "world\n", out.String())
		}()

		go func() {
			defer wg.Done()

			conn, err := l.Accept()
			if !assert.NoError(t, err) {
				return
			}
			if !assert.NotNil(t, conn) {
				return
			}
			defer func() { assert.NoError(t, conn.Close()) }()

			request := make([]byte, 1024)
			n, err := conn.Read(request)
			if !assert.NoError(t, err) {
				return
			}
			assert.Equal(t, "hello\n", string(request)[:n])

			n, err = conn.Write([]byte("world\n"))
			assert.NoError(t, err)
			assert.NotEqual(t, 0, n)
		}()

		wg.Wait()
	})
}

func TestTelnetClientCloseWithoutConnection(t *testing.T) {
	client := NewTelnetClient("127.0.0.1:1", time.Second, io.NopCloser(&bytes.Buffer{}), &bytes.Buffer{})

	require.NoError(t, client.Close())
	require.ErrorIs(t, client.Send(), errNotConnected)
	require.ErrorIs(t, client.Receive(), errNotConnected)
}

func TestTelnetClientConnectRefused(t *testing.T) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	address := l.Addr().String()
	require.NoError(t, l.Close())

	// Слушатель закрыт, поэтому дозвон отклоняется сразу, а не завершается по
	// таймауту; значение таймаута лишь ограничивает попытку.
	client := NewTelnetClient(address, 50*time.Millisecond, io.NopCloser(&bytes.Buffer{}), &bytes.Buffer{})
	require.Error(t, client.Connect())
}

func TestRunInvalidArgs(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		wantErr string
	}{
		{"no args", nil, "usage: go-telnet"},
		{"single arg", []string{"localhost"}, "usage: go-telnet"},
		{"extra arg", []string{"localhost", "4242", "extra"}, "usage: go-telnet"},
		{"non-numeric port", []string{"localhost", "port"}, "invalid port"},
		{"port below range", []string{"localhost", "0"}, "invalid port"},
		{"port above range", []string{"localhost", "65536"}, "invalid port"},
		{"zero timeout", []string{"--timeout=0s", "localhost", "4242"}, "timeout must be positive"},
		{"negative timeout", []string{"--timeout=-1s", "localhost", "4242"}, "timeout must be positive"},
		{"unknown flag", []string{"--bogus", "localhost", "4242"}, "flag provided but not defined"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stdout := &bytes.Buffer{}
			stderr := &bytes.Buffer{}

			err := run(tt.args, io.NopCloser(strings.NewReader("")), stdout, stderr)

			require.Error(t, err)
			require.Contains(t, err.Error(), tt.wantErr)
			require.Empty(t, stdout.String())
		})
	}
}

func TestRunConnectRefused(t *testing.T) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	address := l.Addr().String()
	require.NoError(t, l.Close())

	host, port, err := net.SplitHostPort(address)
	require.NoError(t, err)

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	err = run([]string{"--timeout=1s", host, port}, io.NopCloser(strings.NewReader("")), stdout, stderr)

	require.Error(t, err)
	require.Contains(t, err.Error(), "connection to "+address)
	require.Empty(t, stdout.String())
	require.Empty(t, stderr.String(), "no service message expected before a successful connect")
}

func TestRunConnectTimeout(t *testing.T) {
	// TEST-NET-2 (RFC 5737) зарезервирован для документации и не должен
	// маршрутизироваться, поэтому дозвон может только завершиться ошибкой — по
	// таймауту или из-за недоступности сети, в зависимости от окружения. Тест
	// проверяет, что --timeout передаётся в дозвон и ограничивает попытку, не
	// завися при этом от точного времени.
	const address = "198.51.100.1"
	const timeout = 200 * time.Millisecond

	start := time.Now()
	err := run(
		[]string{"--timeout=" + timeout.String(), address, "123"},
		io.NopCloser(strings.NewReader("")),
		&bytes.Buffer{},
		&bytes.Buffer{},
	)
	elapsed := time.Since(start)

	if err == nil {
		t.Skip("environment can reach TEST-NET-2, cannot exercise a failing dial")
	}
	require.Contains(t, err.Error(), "connection to "+net.JoinHostPort(address, "123"))
	require.Less(t, elapsed, 10*time.Second)
}

func TestRunStdinEOF(t *testing.T) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer func() { assert.NoError(t, l.Close()) }()

	// Собираем результаты, не обращаясь к *testing.T из горутины.
	serverReceived := make(chan string, 1)
	serverDone := make(chan error, 1)
	go func() {
		conn, err := l.Accept()
		if err != nil {
			serverDone <- err
			return
		}
		data, readErr := io.ReadAll(conn)
		closeErr := conn.Close()
		serverReceived <- string(data)
		if readErr != nil {
			serverDone <- readErr
			return
		}
		serverDone <- closeErr
	}()

	host, port, err := net.SplitHostPort(l.Addr().String())
	require.NoError(t, err)

	stdout := &lockedBuffer{}
	stderr := &bytes.Buffer{}
	payload := "I\nwill be\nback!\n"

	err = run([]string{"--timeout=5s", host, port}, io.NopCloser(strings.NewReader(payload)), stdout, stderr)

	require.NoError(t, err, "stdin EOF is a normal shutdown")
	require.Contains(t, stderr.String(), "...Connected to "+net.JoinHostPort(host, port))
	require.Contains(t, stderr.String(), "...EOF")
	require.Empty(t, stdout.String(), "server sent nothing")

	select {
	case data := <-serverReceived:
		require.Equal(t, payload, data)
	case <-time.After(5 * time.Second):
		t.Fatal("server did not receive the payload")
	}
	require.NoError(t, <-serverDone)
}

func TestRunPeerClose(t *testing.T) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer func() { assert.NoError(t, l.Close()) }()

	serverDone := make(chan error, 1)
	go func() {
		conn, err := l.Accept()
		if err != nil {
			serverDone <- err
			return
		}
		_, writeErr := conn.Write([]byte("Bye, client!\n"))
		closeErr := conn.Close()
		if writeErr != nil {
			serverDone <- writeErr
			return
		}
		serverDone <- closeErr
	}()

	host, port, err := net.SplitHostPort(l.Addr().String())
	require.NoError(t, err)

	// Канал stdin остаётся открытым, чтобы Send() не мог опередить закрытие со
	// стороны сервера: программа должна завершиться через путь Receive().
	stdinReader, stdinWriter := io.Pipe()
	defer func() { assert.NoError(t, stdinWriter.Close()) }()

	stdout := &lockedBuffer{}
	stderr := &bytes.Buffer{}

	err = run([]string{"--timeout=5s", host, port}, stdinReader, stdout, stderr)

	require.NoError(t, err, "peer close is a normal shutdown")
	require.Equal(t, "Bye, client!\n", stdout.String())
	require.Contains(t, stderr.String(), "...Connected to "+net.JoinHostPort(host, port))
	require.Contains(t, stderr.String(), "...Connection was closed by peer")
	require.NoError(t, <-serverDone)
}
