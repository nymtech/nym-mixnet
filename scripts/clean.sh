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

# This script should be run from inside the nym-mixnet package directory

logDir="$PWD/logs"

if [ -d $logDir ]
then
    echo "Removing existing logs in the following directory: " $logDir
    rm -rf $logDir
    echo "Creating a new log folder in directory: " $logDir
    mkdir $logDir
else
    echo "Nothing to remove. The logs directory does not exist."
fi

if [ -d $inboxDir ]
then
    echo "Removing existing inboxes in the following directory: " $inboxDir
    rm -rf $inboxDir
    echo "Creating a new inboxes folder in directory: " $inboxDir
    mkdir $inboxDir
else
    echo "Nothing to remove. The inboxes directory does not exist."
fi

function kill_port() {
    PID=$(lsof -t -i:$1)
    echo "Killing process: $PID"
#    kill -TERM ${PID}
    kill -KILL ${PID}
#    kill -TSTP ${PID}
#    kill -CONT ${PID}
}

for var in "$@"
do
    kill_port ${var}
done
