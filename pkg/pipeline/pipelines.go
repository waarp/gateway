package pipeline

import (
	"context"
	"math"
	"sync"

	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
)

const NoLimit = math.MaxUint64

var (
	//nolint:gochecknoglobals //global var is required here since this is a global list
	List = &list{
		m:           map[int64]*Pipeline{},
		limitServer: NoLimit,
		limitClient: NoLimit,
	}

	ErrAlreadyRunning = NewError(types.TeInternal, "transfer is already running")
	ErrLimitReached   = NewError(types.TeExceededLimit, "transfer limit reached")
)

type interruption struct {
	Pause     func(context.Context) error
	Interrupt func(context.Context) error
	Cancel    func(context.Context) error
}

type list struct {
	m     map[int64]*Pipeline
	mutex sync.RWMutex

	limitServer, limitClient uint64
	countServer, countClient uint64
}

func (l *list) add(p *Pipeline) *Error {
	l.mutex.Lock()
	defer l.mutex.Unlock()

	id := p.TransCtx.Transfer.ID

	if _, ok := l.m[id]; ok {
		return ErrAlreadyRunning
	}

	if p.TransCtx.Transfer.IsServer() {
		if l.countServer >= l.limitServer {
			return ErrLimitReached
		}

		l.countServer++
	} else {
		if l.countClient >= l.limitClient {
			return ErrLimitReached
		}

		l.countClient++
	}

	l.m[id] = p

	return nil
}

func (l *list) SetLimits(server, client uint64) {
	l.mutex.Lock()
	defer l.mutex.Unlock()

	if server == 0 {
		server = NoLimit
	}

	if client == 0 {
		client = NoLimit
	}

	l.limitServer = server
	l.limitClient = client
}

func (l *list) GetAvailableOut() uint64 {
	l.mutex.RLock()
	defer l.mutex.RUnlock()

	return l.limitClient - l.countClient
}

func (l *list) Get(id int64) *Pipeline {
	l.mutex.RLock()
	defer l.mutex.RUnlock()

	return l.m[id]
}

func (l *list) remove(id int64) {
	l.mutex.Lock()
	defer l.mutex.Unlock()

	pip, ok := l.m[id]
	if !ok {
		return
	}

	delete(l.m, id)

	if pip.TransCtx.Transfer.IsServer() && l.countServer != 0 {
		l.countServer--
	} else if l.countClient != 0 {
		l.countClient--
	}
}

func (l *list) CancelAll(ctx context.Context) error {
	return l.stopAll(ctx, func(t *model.TransferContext) bool { return true },
		(*Pipeline).Cancel)
}

func (l *list) StopAllFromClient(ctx context.Context, clientID int64,
) error {
	return l.stopAll(ctx, func(t *model.TransferContext) bool {
		return t.Client != nil && t.Client.ID == clientID
	}, (*Pipeline).Interrupt)
}

func (l *list) StopAllFromServer(ctx context.Context, localAgentID int64,
) error {
	return l.stopAll(ctx, func(t *model.TransferContext) bool {
		return t.LocalAgent != nil && t.LocalAgent.ID == localAgentID
	}, (*Pipeline).Interrupt)
}

func (l *list) stopAll(ctx context.Context, cond func(*model.TransferContext) bool,
	halt func(*Pipeline, context.Context) error,
) error {
	var (
		wg      sync.WaitGroup
		stopErr error
		errOnce sync.Once
	)

	defer wg.Wait()

	l.mutex.RLock()
	defer l.mutex.RUnlock()

	for i := range l.m {
		pip := l.m[i]

		if cond(pip.TransCtx) {
			wg.Add(1)

			go func() {
				defer wg.Done()

				if err := halt(pip, ctx); err != nil {
					errOnce.Do(func() { stopErr = err })
				}
			}()
		}
	}

	return stopErr
}

func (l *list) Exists(id int64) bool {
	l.mutex.RLock()
	defer l.mutex.RUnlock()

	_, ok := l.m[id]

	return ok
}

// Reset completely empties the pipelines list. Should only be used in tests.
func (l *list) Reset() {
	l.mutex.Lock()
	defer l.mutex.Unlock()

	l.m = map[int64]*Pipeline{}
}
