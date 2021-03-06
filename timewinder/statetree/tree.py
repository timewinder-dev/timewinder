from hashlib import sha256
from functools import total_ordering
import msgpack

from timewinder.generators import NonDeterministicSet

from typing import Iterable
from typing import List
from typing import Tuple
from typing import Union


@total_ordering
class Hash:
    def __init__(self, b: bytes):
        self.bytes = b

    def hex(self):
        return self.bytes.hex()

    def __eq__(self, other):
        return isinstance(other, Hash) and self.bytes == other.bytes

    def __lt__(self, other):
        return self.bytes < other.bytes

    def __repr__(self) -> str:
        return "Hash(%s)" % self.bytes.hex()

    def __hash__(self):
        return hash(self.bytes)


TreeType = Union[dict, list]
FlatValueType = Union[Hash, str, int, bool, float, None, bytes]
ValidValueType = Union[FlatValueType, NonDeterministicSet]
TreeableType = Union[ValidValueType, dict, list]
HashType = Union[Hash, dict, list]


def _is_deep_type(v) -> bool:
    if isinstance(v, (dict, list, NonDeterministicSet)):
        return True
    return False


def msgpack_ext_default(obj):
    if isinstance(obj, Hash):
        return msgpack.ExtType(1, obj.bytes)
    raise TypeError(f"Unsupported type for serializing tree: {type(obj)}")


_packer = msgpack.Packer(default=msgpack_ext_default)


def _serialize_tree(tree) -> bytes:
    return _packer.pack_map_pairs(sorted(tree.items()))


def _serialize_list(l) -> bytes:
    return _packer.pack(l)


def hash_flat_tree(tree: Union[list, dict]) -> Hash:
    hasher = sha256()
    if isinstance(tree, dict):
        m = _serialize_tree(tree)
    elif isinstance(tree, list):
        m = _serialize_list(tree)
    else:
        raise TypeError("Can only hash dicts or lists")
    hasher.update(m)
    return Hash(hasher.digest())


def non_flat_keys(tree: Union[list, dict]) -> List:
    items: Iterable[Tuple]
    if isinstance(tree, list):
        return [k for k, v in enumerate(tree) if isinstance(v, (dict, list, NonDeterministicSet))]
    else:
        return [k for k in tree if isinstance(tree[k], (dict, list, NonDeterministicSet))]
