"""greet: greetings, written against the greetlib API pinned in requirements.txt."""
from greet.compat import pinned_major

SUPPORTED = {2}  # greetlib majors this code is written for


def greet(name):
    return "Hello, " + name + "!"
