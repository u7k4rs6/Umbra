"""Dynamic dispatch tree-sitter cannot resolve.

Every call below reaches `compute_total` at runtime and none of them names it
at a call site, so no parser can draw the edge. This is the half of the fixture
that exists to be reported as absent rather than confirmed: Umbra's field will
not contain these callers at all, and the report has to say that the field is a
lower bound rather than an answer.
"""

import importlib

import pricing

# 1. A dispatch table. The callee is a value in a dict, selected by a string
#    that arrives at runtime, so the call site reads `handler(...)`.
HANDLERS = {
    "total": pricing.compute_total,
    "line": pricing.line_total,
    "round": pricing.round_money,
}


def dispatch(kind, *args):
    handler = HANDLERS[kind]
    return handler(*args)


# 2. getattr on the module. The name is a string until the moment it is looked
#    up, so the parser sees a call to getattr and nothing else.
def dispatch_by_name(name, *args):
    fn = getattr(pricing, name)
    return fn(*args)


# 3. The same again through importlib, which is how a plugin loader does it.
def dispatch_from_module(module_name, name, *args):
    module = importlib.import_module(module_name)
    return getattr(module, name)(*args)


def invoice_total(subtotal, tax_rate, discount):
    """Reaches compute_total through the table. No parser can see the edge."""
    return dispatch("total", subtotal, tax_rate, discount)


def receipt_total(subtotal, tax_rate, discount):
    """Reaches compute_total through getattr. Same again."""
    return dispatch_by_name("compute_total", subtotal, tax_rate, discount)
