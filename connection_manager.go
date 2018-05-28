package main

import (
	"sync"
)

type ConnectionManager struct {
	connList []interface{}
	lb       LoadBalancer
	lock     sync.RWMutex
}

func (cm *ConnectionManager) AddConnection(conn interface{}) int {
	cm.lock.Lock()
	cm.connList = append(cm.connList, conn)
	connCount := len(cm.connList)
	cm.lb.Update(uint32(connCount-1), uint32(1000))
	cm.lock.Unlock()
	return connCount
}

func (cm *ConnectionManager) DeleteConnection(conn interface{}) int {
	cm.lock.Lock()
	for index, c := range cm.connList {
		if c == conn {
			// delete conneciton
			cm.connList = append(cm.connList[0:index], cm.connList[index+1:]...)
			cm.lb.Delete(uint32(index))
		}
	}
	connCount := len(cm.connList)
	cm.lock.Unlock()
	return connCount
}

func (cm *ConnectionManager) GetConnection() (interface{}, int) {
	cm.lock.RLock()
	connCount := len(cm.connList)
	if connCount == 0 {
		cm.lock.RUnlock()
		return nil, connCount
	}
	index := cm.lb.Get()
	conn := cm.connList[index%connCount]
	cm.lock.RUnlock()
	return conn, connCount
}

func (cm *ConnectionManager) GetAllConnections() []interface{} {
	cm.lock.RLock()
	var connList []interface{}
	connList = append(connList, cm.connList[:]...)
	cm.lock.RUnlock()
	return connList
}

func (cm *ConnectionManager) GetConnectionCount() int {
	cm.lock.RLock()
	connCount := len(cm.connList)
	cm.lock.RUnlock()
	return connCount
}

func (cm *ConnectionManager) UpdateLB(index, weight uint32) {
	cm.lock.Lock()
	cm.lb.Update(index, weight)
	cm.lock.Unlock()
	return
}
