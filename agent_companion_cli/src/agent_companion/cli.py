import argparse
from agent_companion.daemon import AgentCompanionDaemon

def run_cli():
    p = argparse.ArgumentParser()
    p.add_argument("command", choices=["start","status"])
    a = p.parse_args()
    d = AgentCompanionDaemon()
    if a.command=="start": d.run()
    else: d.print_status()
