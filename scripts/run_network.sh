#// Copyright 2018 The Loopix-Messaging Authors
#//
#// Licensed under the Apache License, Version 2.0 (the "License");
#// you may not use this file except in compliance with the License.
#// You may obtain a copy of the License at
#//
#//      http://www.apache.org/licenses/LICENSE-2.0
#//
#// Unless required by applicable law or agreed to in writing, software
#// distributed under the License is distributed on an "AS IS" BASIS,
#// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
#// See the License for the specific language governing permissions and
#// limitations under the License.

#!/bin/bash

echo "Press CTRL-C to stop."
rm -rf "$PWD/pki/database.db"

logDir="$PWD/logs"

if [ -d $logDir ]
then
    echo "Logging directory already exists"
else
    mkdir $logDir
    echo "Created logging directory"
fi

NUMMIXES=${1:-3} # Set $NUMMIXES to default of 3, but allow the user to set other values if desired

for (( j=0; j<$NUMMIXES; j++ ))
do
    $PWD/build/loopix-mixnode run --id "Mix$j" --port $((9980+$j)) >> logs/bash.log &
    sleep 1
done

sleep 1
$PWD/build/loopix-provider run --id Provider --port 9997 >> logs/bash.log

# trap call ctrl_c()
trap ctrl_c SIGINT SIGTERM SIGTSTP
function ctrl_c() {
        echo "** Trapped SIGINT, SIGTERM and SIGTSTP"
        for (( j=0; j<$NUMMIXES; j++ ));
        do
            kill_port $((9980+$j))
        done
}

function kill_port() {
    PID=$(lsof -t -i:$1)
    echo "$PID"
    kill -TERM $PID || kill -KILL $PID
}
