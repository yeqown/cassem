## cassemd

To start the `server` and it's daemon process, components in `server` include: 
* `RESTful HTTP`
* `Authorize` middleware
* `coordinator`
* `Cache` middleware
* `Watcher` to watch containers' changes. 
* `Persistence` to persist `cassem's` data.

### Get started

Now, you can start the `cassemd` server as following command:

```sh
# @yeqown@gmail.com
# @cassemd

./cassemd \
	-c ./configs/cassem.example.toml \		# config file path
	--id="2e422fdf" \ 						# nodeID of cassemd which is unique
	--raft-base="./debugdata/2e422fdf"  \ 	# raft base directory to store
	--http-listen="127.0.0.1:2021" \		# cassemd restful HTTP address
	--bind="127.0.0.1:3021" \				# address for raft protocol to communicate to each other
	--join=""								# address to send join cluster request
```

```sh
./cassemd \
	-c ./configs/cassem.example.toml \
	--id="2e422fdf" \
	--raft-base="./debugdata/2e422fdf"  \
	--http-listen="127.0.0.1:2021" \
	--bind="127.0.0.1:3021" \
	--join=""

./cassemd -c ./configs/cassem.example.toml \
	--id=b6a77ac2 \
	--raft-base="./debugdata/b6a77ac2" \
	--http-listen="127.0.0.1:2022" \
	--bind="127.0.0.1:3022" \
	--join="127.0.0.1:2021"

./cassemd -c ./configs/cassem.example.toml \
	--id="a035b428" \
	--raft-base="./debugdata/a035b428" \
	--http-listen="127.0.0.1:2023" \
	--bind="127.0.0.1:3023" \
	--join="127.0.0.1:2021"
```

then you'll get the following content:

```sh
# started output
INF _ts="1614235413" msg="Daemon: HTTP server loaded"
2021-02-25T14:43:33.856+0800 [INFO]  raft: initial configuration: index=0 servers=[]
2021-02-25T14:43:33.857+0800 [INFO]  raft: entering follower state: follower="Node at 127.0.0.1:3021 [Follower]" leader=
DBG _ts="1614235413" msg="server running on: :2021"
[GIN-debug] Listening and serving HTTP on :2021
2021-02-25T14:43:35.609+0800 [WARN]  raft: heartbeat timeout reached, starting election: last-leader=
2021-02-25T14:43:35.610+0800 [INFO]  raft: entering candidate state: node="Node at 127.0.0.1:3021 [Candidate]" term=2
2021-02-25T14:43:35.701+0800 [DEBUG] raft: votes: needed=1
2021-02-25T14:43:35.702+0800 [DEBUG] raft: vote granted: from=JmpOn5di term=2 tally=1
2021-02-25T14:43:35.702+0800 [INFO]  raft: election won: tally=1
2021-02-25T14:43:35.702+0800 [INFO]  raft: entering leader state: leader="Node at 127.0.0.1:3021 [Leader]"

```