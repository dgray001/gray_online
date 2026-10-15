#!/usr/bin/env bash
dir="$(dirname "$0")"
for s in eval_d_crush eval_e_mixed eval_f_snowball eval_g_nomad eval_h_dynamic eval_i_forward eval_j_mopup eval_k_garrison; do
  echo "========================================"
  echo "RUNNING $s"
  echo "========================================"
  "$dir/sim" -quiet "$s"
done
