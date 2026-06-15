import re
from typing import Dict, List, Optional, Tuple
from pathlib import Path
from dataclasses import dataclass
from enum import Enum

class Severity(Enum):
    INFO = "INFO"
    WARNING = "WARNING"
    ERROR = "ERROR"
    CRITICAL = "CRITICAL"

@dataclass
class LogEvent:
    severity: Severity
    message: str
    file_path: Path
    line_number: Optional[int] = None
    timestamp: Optional[str] = None
    stack_trace: Optional[str] = None
    category: Optional[str] = None

class LogParser:
    """Intelligent log parser that extracts errors, warnings, and meaningful events."""

    NOISE_MESSAGE_FRAGMENTS = (
        "bad parameter or other api misuse",
        "sqlite",
        "database is locked",
    )

    SKIP_PATH_FRAGMENTS = (
        "agent_companion_data",
        "/terminals/",
        "agent-transcripts",
        "/.hermes/",
        "memory.db",
    )

    @classmethod
    def should_skip_path(cls, file_path: Path) -> bool:
        s = str(file_path).lower()
        return any(frag in s for frag in cls.SKIP_PATH_FRAGMENTS)

    @classmethod
    def is_noise_message(cls, message: str) -> bool:
        m = (message or "").lower()
        return any(frag in m for frag in cls.NOISE_MESSAGE_FRAGMENTS)

    ERROR_PATTERNS = [
        r'(?i)error\s*[:\-]?\s*(.+)',
        r'(?i)exception\s*[:\-]?\s*(.+)',
        r'(?i)failed\s*[:\-]?\s*(.+)',
        r'(?i)fatal\s*[:\-]?\s*(.+)',
        r'(?i)traceback',
        r'(?i)assert(?:ion)?\s+failed',
    ]
    
    WARNING_PATTERNS = [
        r'(?i)warning\s*[:\-]?\s*(.+)',
        r'(?i)deprecated\s*[:\-]?\s*(.+)',
        r'(?i)caution\s*[:\-]?\s*(.+)',
    ]
    
    STACK_TRACE_PATTERNS = [
        r'^\s+at\s+',
        r'^\s+File\s+"',
        r'^\s+in\s+\w+',
        r'^\s+\d+:\s+',
    ]
    
    def __init__(self):
        self.error_regex = [re.compile(p) for p in self.ERROR_PATTERNS]
        self.warning_regex = [re.compile(p) for p in self.WARNING_PATTERNS]
        self.stack_regex = [re.compile(p) for p in self.STACK_TRACE_PATTERNS]
    
    def parse_file(self, file_path: Path) -> List[LogEvent]:
        """Parse a file and extract all log events with context."""
        if self.should_skip_path(file_path):
            return []
        events = []
        try:
            content = file_path.read_text(encoding='utf-8', errors='ignore')
            lines = content.splitlines()
            
            for i, line in enumerate(lines):
                for pattern in self.error_regex:
                    match = pattern.search(line)
                    if match:
                        stack_trace = self._extract_stack_trace(lines, i + 1)
                        # Extract 3 lines of context around the error
                        start = max(0, i - 2)
                        end = min(len(lines), i + 3)
                        context = "\n".join(lines[start:end])
                        
                        message = match.group(1) if match.groups() else line.strip()
                        if self.is_noise_message(message):
                            continue
                        events.append(LogEvent(
                            severity=Severity.ERROR,
                            message=message,
                            file_path=file_path,
                            line_number=i + 1,
                            stack_trace=stack_trace,
                            category=f"Context:\n```\n{context}\n```"
                        ))
                        break
        except Exception as e:
            events.append(LogEvent(severity=Severity.WARNING, message=f"Parse error: {e}", file_path=file_path))
        return events
    
    def _extract_stack_trace(self, lines: List[str], start_idx: int) -> Optional[str]:
        """Extract stack trace starting from given line index."""
        stack_lines = []
        
        for i in range(start_idx, min(start_idx + 20, len(lines))):
            line = lines[i]
            is_stack = any(pattern.match(line) for pattern in self.stack_regex)
            
            if is_stack:
                stack_lines.append(line)
            elif stack_lines:
                break
        
        return '\n'.join(stack_lines) if stack_lines else None
    
    def categorize_file(self, file_path: Path) -> str:
        """Categorize a file by type."""
        suffix = file_path.suffix.lower()
        
        if suffix in ['.log', '.txt']:
            return 'log'
        elif suffix in ['.py', '.js', '.ts', '.tsx', '.jsx', '.java', '.cpp', '.c', '.go', '.rs']:
            return 'code'
        elif suffix in ['.json', '.yaml', '.yml', '.toml', '.ini', '.conf']:
            return 'config'
        elif suffix in ['.md', '.rst', '.txt']:
            return 'documentation'
        else:
            return 'other'
    
    def extract_failure_signature(self, event: LogEvent) -> str:
        """Generate a unique signature for a failure to detect duplicates."""
        sig_parts = [
            event.message[:100],
            str(event.file_path.name),
            str(event.severity.value)
        ]
        return '|'.join(sig_parts)
    
    def analyze_code_change(self, file_path: Path) -> Dict[str, any]:
        """Analyze a code file change for meaningful context."""
        try:
            content = file_path.read_text(encoding='utf-8', errors='ignore')
            lines = content.splitlines()
            
            analysis = {
                'line_count': len(lines),
                'has_imports': False,
                'has_errors': False,
                'has_todos': False,
                'complexity_hint': 'low'
            }
            
            for line in lines:
                line_lower = line.lower()
                if 'import ' in line_lower or 'require(' in line_lower:
                    analysis['has_imports'] = True
                if 'error' in line_lower or 'exception' in line_lower:
                    analysis['has_errors'] = True
                if 'todo' in line_lower or 'fixme' in line_lower:
                    analysis['has_todos'] = True
            
            if len(lines) > 500:
                analysis['complexity_hint'] = 'high'
            elif len(lines) > 200:
                analysis['complexity_hint'] = 'medium'
            
            return analysis
            
        except Exception:
            return {'error': 'Could not analyze'}
