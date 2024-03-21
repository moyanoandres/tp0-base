#!/bin/bash
CONTAINER_NAME="nc_test"
NETWORK_NAME="tp0_testing_net"
SERVER="server"
PORT="12345"

TEST_MSG="Testing 123321"

#Run test container in detached mode
CONTAINER_ID=$(docker run -d --rm --name $CONTAINER_NAME --network $NETWORK_NAME alpine sh -c "tail -f /dev/null")
if [ $? -ne 0 ]; then
    exit #Exit the script if the Docker run command failed
fi

#Send message to server using netcat
echo "Sending test message to server: '$TEST_MSG'"
response=$(docker exec $CONTAINER_ID sh -c "echo \"$TEST_MSG\" | nc $SERVER $PORT")

#Check if message was successfully sent
if [ $? -eq 0 ]; then
    #Check server response
    if [ "$response" == "$TEST_MSG" ]; then
        echo "Echo Server up and running!"
    else
        echo "Unexpected response from server: $response"
    fi
else
    echo "Error. Either the wrong address was provided or server is not currently running"
fi

docker stop $CONTAINER_ID