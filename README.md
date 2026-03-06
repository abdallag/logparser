# logparser

## Build

```bash
cd /home/aabdulmonem/nime/logparser

# Build binary
go build -o logparser ./cmd/logparser

# Or use Make
make build
```

## Run

```bash
# Megatron-style log
./logparser ../55B_hybrid_moe_25T_phase2_1745223_date_26-02-21_time_01-31-49.log

# NeMo-style log
./logparser ../log-general_cs_infra-infra.pretrain_deepseek_v3_bf16_gpus8_tp1_pp1_cp1_vp1_ep8_mbs1_gbs64_1492926_0.out

# Table output
./logparser -format table ../55B_hybrid_moe_25T_phase2_1745223_date_26-02-21_time_01-31-49.log

# Run without building (go run)
go run ./cmd/logparser ../55B_hybrid_moe_25T_phase2_1745223_date_26-02-21_time_01-31-49.log
```
