# memcachedump
memcached dump &amp; restore tool

usage
-----

ex:

```
memcached -address 10.0.0.1:11211 dump >dump.json
```

```
memcached -address 10.0.0.2:11211 restore <dump.json
```

```
memcached -address 10.0.0.1:11211 dump | memcached -address 10.0.0.2:11211 restore
```
