# rscore CLI

## Initialization
```bash
brew install go
go mod init github.com/zhouselena/resilienceScore
go get -u github.com/spf13/cobra@latest
```

## Run (with build)
```bash
bash run-rscore.sh
```

## Run tests

## Notes

```go

/* STACK */

var stack []string

// Push
stack = append(stack, "item") 

// Pop (with basic empty check)
if len(stack) > 0 {
    top := stack[len(stack)-1] // Get last item
    stack = stack[:len(stack)-1] // Remove last item
}

// isEmpty
len(stack) == 0

/* QUEUE */

var queue []string

// Enqueue
queue = append(queue, "item")

// Dequeue (with basic empty check)
if len(queue) > 0 {
    front := queue[0]           // Get first item
    queue = queue[1:]           // Remove first item
}

// isEmpty
len(queue) == 0
```