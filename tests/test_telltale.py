import timewinder
import pytest

from tests.test_helpers import A


def test_simple_eval():
    @timewinder.step
    def t(state, m):
        if m.foo == "a":
            m.foo = "b"
            return
        m.foo = "a"

    a = A()

    ev = timewinder.Evaluator(
        objects=[a],
        threads=[t(a)],
    )

    ev.evaluate(steps=4)


def test_multi_eval():
    @timewinder.step
    def t(state, m):
        m.foo = "b"

    @timewinder.step
    def x(state, m):
        assert m.foo == "b"
        m.foo = "a"

    a = A()

    ev = timewinder.Evaluator(
        objects=[a],
        threads=[timewinder.FuncProcess(t(a), x(a))],
    )

    ev.evaluate(steps=4)
    ev._print_state_space()
    assert ev.stats.states == 3


def test_subcall_onestate():
    @timewinder.step
    def t(state, m):
        m.foo = "b"
        x(m)

    # Intentionally not a thread -- so t calls it in one step
    def x(m):
        assert m.foo == "b"
        m.foo = "a"

    a = A()

    ev = timewinder.Evaluator(
        objects=[a],
        threads=[t(a)],
    )

    ev.evaluate(steps=4)
    assert ev.stats.states == 2


def test_assert_fail():
    @timewinder.step
    def t(state, m):
        m.foo = "b"

    @timewinder.step
    def x(state, m):
        assert m.foo == "b"

    a = A()

    ev = timewinder.Evaluator(
        objects=[a],
        threads=[t(a), x(a)],
    )

    with pytest.raises(AssertionError):
        ev.evaluate(steps=4)
