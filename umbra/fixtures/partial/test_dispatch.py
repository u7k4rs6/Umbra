"""Tests that reach compute_total only through the dispatch table.

These are the tests Umbra cannot select, because selection walks graph edges
and there are no edges to walk. A run against this fixture must not report a
clean selection: the sweep is the only thing that covers these.
"""

from billing import invoice
from dispatch import dispatch, dispatch_by_name, invoice_total, receipt_total


def test_invoice_total():
    assert invoice_total(100.0, 0.1, 5.0) == 105.0


def test_receipt_total():
    assert receipt_total(100.0, 0.1, 5.0) == 105.0


def test_dispatch_table_reaches_compute_total():
    assert dispatch("total", 100.0, 0.0, 0.0) == 100.0


def test_dispatch_by_name_reaches_compute_total():
    assert dispatch_by_name("compute_total", 100.0, 0.0, 0.0) == 100.0


def test_invoice_through_the_rate_card():
    assert invoice(100.0, 5.0) == 105.0
