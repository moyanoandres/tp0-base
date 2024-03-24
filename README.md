# TP0: Docker + Comunicaciones + Concurrencia

## Ej1:

El script creado para la creación del DockerCompose con clientes configurables es compose_maker.py 

### Instrucción de uso: 

Desde el root del proyecto ejecutar:
```bash
python compose_maker.py [n]
```

siendo [n] la cantidad de clientes que se desee configurar (1 si no se provee).

## Ej2:

Se definen volumes en la DockerCompose file que mapean files en el host system a paths en los containers para que estos puedan acceder a su configuración sin que se deba buildear una nueva imagen tras cambiar la configuración de alguno.    

Ejemplo que mapea ./server/config.ini en el host a /config.ini en el container del servidor:    
```
volumes:    
    - ./server/config.ini:/config.ini
```

## Ej3:

El script creado para verificar el funcionamiento del servidor como Echo Server es nc_test.sh   

Este script crea un container que se conecta a la network en la que está conectado el servidor y ejecuta "tail -f /dev/null" para que continúe en ejecución indefinidamente y así poder ejecutar netcat dentro de él.   

Este container envía el mensaje a la dirección del servidor usando el comando "nc" y el script revisa que la respuesta recibida sea igual al mensaje enviado.   

Una vez hecho esto el container se detiene y se elimina.    

Nota: actualizar los valores de las variables al inicio del script en caso de cambiar la dirección del servidor y/o el nombre de la red.

## Ej4:

### Servidor:

El servidor mantiene en su estado el socket asociado al cliente (si hubiera una conexion activa), y por supuesto el socket para aceptar conexiones. Cuando recibe SIGTERM, pone en True su flag _shutting_down e invoca shutdown() en los sockets activos. Esto resulta en que si el shutdown se efectúa en una operación bloqueante, como podría ser el proceso de aceptar una nueva conexión, esta operación devuelva error y esto se pueda catchear y proceder con el graceful shutdown. El shutdown procede con el llamado a Close() sobre los sockets que estén abiertos y la salida del loop de ejecución. 

### Cliente:

El cliente mantiene en su estado al socket que corresponde a su conexión. Antes de inicial el loop de ejecución, se crea un channel para la señal SIGTERM y se inicia una goroutine esperando la llegada de la señal al channel. Si se recibe la señal, el cliente setea su flag shuttingDown en true y si cierra el socket si estuviera abierto. Nuevamente, si esto sucediera durante operaciones del socket, el retorno de err sería distinto de nil, y se procede con el shutdown. El programa avanza hasta el final del ciclo acutal y cierra el socket si fuera necesatrio, pero no inicia un nuevo ciclo y finaliza. 

## Ej5:

Para el nuevo caso de uso de lotería, los mensajes que envían los clientes (que ahora cumplen el rol de agencias) y el servidor (que ahora cumplen el rol de lotería nacional) son de acuerdo al protocolo que definí:  

[HEADER],[PAYLOAD]  

donde [HEADER] := [TIPO_MENSAJE],[TAMAÑO_PAYLOAD]   

El header de los mensajes tiene un tamaño fijo de 8 bytes, 3 para el tipo de mensaje, una coma, y 4 para el tamaño del payload  

TIPO_MENSAJE puede ser BET o ACK. BET corresponde a un cliente solicitando el registro de una apuesta y ACK es la confirmación del servidor al cliente de que la apuesta se almacenó correctamente. 

El formato del payload dependerá del tipo de mensaje, para un mensaje BET es:   [agencia],[betID],[nombre],[apellido],[DNI],[Fecha],[Numero]    

Tanto la aplicacion servidor como cliente tienen un nuevo archivo common/communication.xx, con la responsabilidad de manejar la interpretación de los mensajes según el protocolo, y de leer y escribir adecuadamente todo el mensaje.   

