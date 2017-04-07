## Notify List
```go
type notifyList struct {
	wait uint32         // 下个等待者的 ticket
	notify uint32       // 下个唤醒的 ticket

	lock mutex          // 等待队列相关结构
	head *sudog
	tail *sudog
}

func less(a, b uint32) bool {
	return int32(a-b) < 0
}

func notifyListAdd(l *notifyList) uint32 {
    // 增加计数，并返回原有计数
    // 获取当前等待 ticket
    // 并调整下个 ticket
	return atomic.Xadd(&l.wait, 1) - 1
}

func notifyListWait(l *notifyList, t uint32) {
	lock(&l.lock)

    // 如果已经通知过 ticket，不需要等待
	if less(t, l.notify) {
		unlock(&l.lock)
		return
	}

    // 没有通知到，入队
	s := acquireSudog()
	s.g = getg()
	s.ticket = t            // 设置 ticket
	s.releasetime = 0
	t0 := int64(0)
	if blockprofilerate > 0 {
		t0 = cputicks()
		s.releasetime = -1
	}
	
	// 入队尾
	if l.tail == nil {
		l.head = s
	} else {
		l.tail.next = s
	}
	l.tail = s
	
	// 当前 goroutine 进入等待状态，并解锁
	goparkunlock(&l.lock, "semacquire", traceEvGoBlockCond, 3)
	
	// 恢复调度后执行
	if t0 != 0 {
		blockevent(s.releasetime-t0, 2)
	}
	releaseSudog(s)
}

func notifyListNotifyAll(l *notifyList) {
	// 不需要通知
	if atomic.Load(&l.wait) == atomic.Load(&l.notify) {
		return
	}

    // 注意这里的操作：
    // 加锁，处理链表，解锁后，链表内容不会发生变化！
	lock(&l.lock)
	s := l.head
	l.head = nil
	l.tail = nil

	// 修改 notify ticket 为最大号
	atomic.Store(&l.notify, atomic.Load(&l.wait))
	unlock(&l.lock)

	// 激活全部等待者
	for s != nil {
		next := s.next
		s.next = nil
		readyWithTime(s, 4)
		s = next
	}
}

func notifyListNotifyOne(l *notifyList) {
	if atomic.Load(&l.wait) == atomic.Load(&l.notify) {
		return
	}

	lock(&l.lock)

	// 加锁后再次检查
	t := l.notify
	if t == atomic.Load(&l.wait) {
		unlock(&l.lock)
		return
	}

    // 通知序号加一，用于下次通知
	atomic.Store(&l.notify, t+1)
	
	// 遍历等待队列，找到 ticket == t 的节点
	for p, s := (*sudog)(nil), l.head; s != nil; p, s = s, s.next {
		if s.ticket == t {
		    // s 出队
			n := s.next
			if p != nil {
				p.next = n
			} else {
				l.head = n
			}
			
			if n == nil {
				l.tail = p
			}
			unlock(&l.lock)     // 不需要锁时，尽早释放！
			
			s.next = nil
			readyWithTime(s, 4)
			return
		}
	}
	unlock(&l.lock)
}
```
