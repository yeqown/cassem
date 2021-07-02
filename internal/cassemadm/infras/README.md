## cassemadm concepts

```
cassemdb has key structure like this:

root
	/elements
	|	|-app-1
	|	|	|-env-1
	|	|	|	|- elt-1
	|	|	|		|-metadata
	|	|	|		|-v1
	|	|	|		|-v2
	|	|	|-env-2
	|	|-app-2
	/apps
	|	|-app1
	|		|-metadata
	|	|-app2
	|		|-metadata
	/envs/
	|	|-env-1
	|		|-metadata
	|	|-env-1
	/locks
		|- appid-env-elt (lock metadata)
```