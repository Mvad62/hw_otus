package hw06pipelineexecution

type (
	In  = <-chan interface{}
	Out = In
	Bi  = chan interface{}
)

type Stage func(in In) (out Out)

func ExecutePipeline(in In, done In, stages ...Stage) Out {
	out := forward(done, in)
	for _, stage := range stages {
		out = stage(out)
		out = forward(done, out)
	}
	return out
}

// forward оборачивает входной канал и перенаправляет значения в выходной канал.
// Если канал done закрыт, он прекращает перенаправление и вычитывает входной канал,
// чтобы разблокировать предыдущий стейдж, после чего закрывает выходной.
func forward(done In, in In) Out {
	out := make(Bi)
	go func() {
		defer close(out)
		for {
			select {
			case <-done:
				// Вычитываем вход отдельно, чтобы закрыть out без ожидания upstream.
				go drain(in)
				return
			case v, ok := <-in:
				if !ok {
					return
				}
				select {
				case out <- v:
				case <-done:
					go drain(in)
					return
				}
			}
		}
	}()
	return out
}

func drain(in In) {
	for {
		if _, ok := <-in; !ok {
			return
		}
	}
}
