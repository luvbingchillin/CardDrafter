import sys
from pathlib import Path

# Ensures generated sibling modules (analytics_pb2) can be imported seamlessly
sys.path.insert(0, str(Path(__file__).parent.resolve()))
