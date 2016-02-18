# memcachedump
[![Circle CI](https://circleci.com/gh/masahide/memcachedump/tree/master.svg?style=svg)](https://circleci.com/gh/masahide/memcachedump/tree/master)

memcached dump/restore tool

usage
-----

ex:

```
 $ memcachedump -address 10.0.0.1:11211 dump > dump.json
 $ memcachedump -address 10.0.0.2:11211 restore < dump.json
 $ memcachedump -address 10.0.0.1:11211 dump | memcachedump -address 10.0.0.2:11211 restore
```
