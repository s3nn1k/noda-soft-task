package main

import (
	"fmt"
	"strings"
	"sync"
	"time"
)

// ЗАДАНИЕ: сделать из плохого кода хороший и рабочий - as best as you can.
// Важно сохранить логику появления ошибочных тасков.
// Важно оставить асинхронные генерацию и обработку тасков.
// Сделать правильную мультипоточность обработки заданий.
// Обновленный код отправить через pull-request в github
// Как видите, никаких привязок к внешним сервисам нет - полный карт-бланш на модификацию кода.

// Мы даем тестовое задание чтобы:
// * уменьшить время технического собеседования - лучше вы потратите пару часов в спокойной домашней обстановке, чем будете волноваться, решая задачи под взором наших ребят;
// * увеличить вероятность прохождения испытательного срока - видя сразу стиль и качество кода, мы можем быть больше уверены в выборе;
// * снизить число коротких собеседований, когда мы отказываем сразу же.

// Выполнение тестового задания не гарантирует приглашение на собеседование, т.к. кроме качества выполнения тестового задания, оцениваются и другие показатели вас как кандидата.
// Мы не даем комментариев по результатам тестового задания. Если в случае отказа вам нужен наш комментарий по результатам тестового задания, то просим об этом написать вместе с откликом.

const (
	workTime          = 10 * time.Second // Приложение должно генерировать таски 10 сек
	sendingInterval   = 3 * time.Second  // Каждые 3 секунды должно выводить в консоль результат всех обработанных к этому моменту тасков (отдельно успешные и отдельно с ошибками)
	producingInterval = 1 * time.Nanosecond
)

type Task struct {
	id                  int
	createdAtTimestamp  string // время создания
	finishedAtTimestamp string // время выполнения
	result              []byte
}

func (t *Task) String() string {
	return fmt.Sprintf("Id: %d, Start: %s, Finish: %s, Result: %s", t.id, t.createdAtTimestamp, t.finishedAtTimestamp, string(t.result))
}

// Приложение эмулирует получение и обработку неких тасков. Пытается и получать, и обрабатывать в многопоточном режиме.
func main() {
	tasksChan := make(chan Task, 10)
	doneChan := make(chan Task, 10)
	errChan := make(chan Task, 10)

	go produceTasks(tasksChan, workTime, producingInterval)

	go processTasks(tasksChan, doneChan, errChan)

	printTasks(doneChan, errChan, sendingInterval, workTime)
}

// генерация тасков
func produceTasks(tasksChan chan<- Task, workTime time.Duration, producingInterval time.Duration) {
	start := time.Now().Add(workTime).Unix()

	for time.Now().Unix() < start {
		createdAtTimestamp := time.Now().Format(time.RFC3339)

		if time.Now().Nanosecond()%2 > 0 { // вот такое условие появления ошибочных тасков
			createdAtTimestamp = "Some error occured"
		}

		tasksChan <- Task{createdAtTimestamp: createdAtTimestamp, id: int(time.Now().Unix())} // передаем таск на выполнение

		time.Sleep(producingInterval)
	}

	close(tasksChan)
}

// получение и обработка тасков
func processTasks(tasksChan <-chan Task, doneChan chan<- Task, errChan chan<- Task) {
	sortingTasks := &sync.WaitGroup{}

	for task := range tasksChan {
		start, _ := time.Parse(time.RFC3339, task.createdAtTimestamp)

		if start.After(time.Now().Add(-20 * time.Second)) {
			task.result = []byte("task has been successed")
		} else {
			task.result = []byte("something went wrong")
		}

		task.finishedAtTimestamp = time.Now().Format(time.RFC3339Nano)

		sortingTasks.Add(1)

		go sortTask(task, doneChan, errChan, sortingTasks)
	}

	sortingTasks.Wait()

	close(doneChan)
	close(errChan)
}

// сортировка тасков
func sortTask(task Task, doneChan chan<- Task, errChan chan<- Task, sortingTasks *sync.WaitGroup) {
	if strings.Contains(string(task.result), "success") {
		doneChan <- task
	} else {
		errChan <- task
	}

	sortingTasks.Done()
}

// вывод тасков в консоль
func printTasks(doneChan <-chan Task, errChan <-chan Task, sendingInterval time.Duration, workTime time.Duration) {
	start := time.Now().Add(workTime).Unix()
	printingFuncs := &sync.WaitGroup{}

	for time.Now().Unix() < start {
		printingFuncs.Add(2)

		go fetchTasks(doneChan, sendingInterval, "Done Tasks:", printingFuncs)

		go fetchTasks(errChan, sendingInterval, "Errors:", printingFuncs)

		printingFuncs.Wait()
	}
}

func fetchTasks(tasksChan <-chan Task, sendingInterval time.Duration, prefix string, printingFuncs *sync.WaitGroup) {
	ticker := time.NewTicker(sendingInterval)
	var complete bool

	tasks := []Task{}
	mu := sync.Mutex{}

	for {
		if complete {
			sendTasks(tasks, prefix)

			printingFuncs.Done()
			return
		}

		select {
		case task, ok := <-tasksChan:
			if !ok {
				complete = true
			}

			go func() {
				mu.Lock()
				tasks = append(tasks, task)
				mu.Unlock()
			}()

		case <-ticker.C:
			sendTasks(tasks, prefix)

			tasks = []Task{}
		}
	}
}

func sendTasks(tasks []Task, prefix string) {
	text := prefix + "\n"

	for _, task := range tasks {
		text += task.String() + "\n"
	}

	fmt.Print(text)
}
