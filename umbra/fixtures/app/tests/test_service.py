"""Totals and rounding. This file calls compute_total directly."""

from app.models import LineItem
from app.service import apply_tax, compute_total, round_money


def test_rounding():
    items = [LineItem(sku="a", quantity=3, unit_price=3.33)]
    total = compute_total(items)
    assert total == 10.79


def test_negative():
    items = [LineItem(sku="a", quantity=-1, unit_price=5.00)]
    total = compute_total(items)
    assert total == -5.40


def test_empty_is_zero():
    assert compute_total([]) == 0.0


def test_round_money_places():
    assert round_money(1.005) == 1.01


def test_apply_tax_adds_rate():
    assert apply_tax(100.0) == 108.0
