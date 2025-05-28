// internal/tasks/interface.go
package tasks

type TaskManagerInterface interface {
	RegisterTasks() error
}