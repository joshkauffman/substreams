package loop

import "context"

// loop is the micro framework for the Scheduler's event loop,
// heavily inspired by BubbleTea, which we use for the substreams GUI.

type EventLoop struct {
	ctx        context.Context
	msgs       chan Msg
	updateFunc func(msg Msg) Cmd
}

func NewEventLoop(updateFunc func(msg Msg) Cmd) EventLoop {
	return EventLoop{
		msgs:       make(chan Msg, 1000),
		updateFunc: updateFunc,
	}
}

func (l *EventLoop) Run(ctx context.Context, initCmd Cmd) (err error) {
	l.ctx = ctx
	cmds := make(chan Cmd, 1000)
	if initCmd != nil {
		cmds <- initCmd
	}
	// main execution loop
	done := l.handleCommands(cmds)
loop:
	for {
		select {
		case <-l.ctx.Done():
			err = l.ctx.Err()
			break loop
		case msg := <-l.msgs:
			if quit, ok := msg.(QuitMsg); ok {
				err = quit.err
				break loop
			}

			cmd := l.update(msg, cmds)
			if cmd == nil {
				continue
			}
			go func() {
				msg := cmd()
				l.Send(msg)
			}()
		}
	}
	<-done
	return
}

func (l *EventLoop) Send(msg Msg) {
	select {
	case <-l.ctx.Done():
	case l.msgs <- msg:
	}
}

func (l *EventLoop) update(msg Msg, cmds chan Cmd) (out Cmd) {
	switch msg := msg.(type) {
	case BatchMsg:
		for _, cmd := range msg {
			cmds <- cmd
		}
		return nil

	case SequenceMsg:
		go func() {
			// Execute commands one at a time, in order.
			for _, cmd := range msg {
				l.Send(cmd())
			}
		}()
	}

	return l.updateFunc(msg)
}

func (l *EventLoop) handleCommands(cmds chan Cmd) chan struct{} {
	ch := make(chan struct{})

	go func() {
		defer close(ch)

		for {
			select {
			case <-l.ctx.Done():
				return

			case cmd := <-cmds:
				if cmd == nil {
					continue
				}

				go func() {
					msg := cmd() // this can be long.
					l.Send(msg)
				}()
			}
		}
	}()

	return ch
}
