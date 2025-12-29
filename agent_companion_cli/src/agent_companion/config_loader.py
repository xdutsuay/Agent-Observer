import yaml, json, os
from pathlib import Path

def load_config():
    with open("config/agent_companion.yaml") as f:
        cfg = yaml.safe_load(f)
    cfg["watch_paths"]=[str(Path(p).expanduser()) for p in cfg["watch_paths"]]
    cfg["output_root"]=str(Path(cfg["output_root"]).expanduser())
    return cfg

def load_heuristics():
    with open("config/heuristics.json") as f:
        return json.load(f)
