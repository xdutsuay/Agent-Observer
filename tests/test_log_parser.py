from pathlib import Path

from agent_memory_mcp.log_parser import LogParser, Severity


def test_parse_error_line(tmp_path):
    log = tmp_path / "app.log"
    log.write_text("INFO ok\nERROR: database connection failed\n  at foo.bar:10\n")
    parser = LogParser()
    events = parser.parse_file(log)
    assert any(e.severity == Severity.ERROR for e in events)


def test_categorize_code():
    parser = LogParser()
    assert parser.categorize_file(Path("main.py")) == "code"


def test_noise_filter():
    parser = LogParser()
    assert parser.is_noise_message("bad parameter or other API misuse")
    assert parser.should_skip_path(Path("/Users/x/.cursor/projects/foo/terminals/1.txt"))
