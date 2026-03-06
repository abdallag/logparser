# logparser

## Build

# Build binary
go build -o logparser ./cmd/logparser

# Or use Make
make build
```

## Run

```bash
# Megatron-style log
./logparser ../log-mega.log

# NeMo-style log
./logparser ../log-nemo.out

# Table output
./logparser -format table ../log-mega.log

# Run without building (go run)
go run ./cmd/logparser ../log-nemo.log
```
