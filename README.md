# TP0: Docker + Comunicaciones + Concurrencia

# IMPORTANTE

Del ejercicio 6 en adelante había un error si se elegía un batchsize muy chico para archivos muy grandes, o un batchsize muy grande para cualquier tipo de archivo. De este error me di cuenta más adelante en el TP y lo corregí. El protocolo corregido está [acá](#protocolo-versión-final). Descontando el aumento del tamaño de los headers, todo lo que dice en los respectivos apartados de los ejercicios no tuvo cambios.

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

## Ej 6:

El protocolo de envío y recepción de mensajes fue cambiado para soportar mensajes BET con un batch de apuestas en vez de una sola, el nuevo formato para BET es:    

[HEADER];[PAYLOAD]  

donde [HEADER] := BET[PAYLOAD_SIZE][BATCH_SIZE][BATCH_ID]   
con un tamaño fijo de 13 bytes, 3 del tipo de mensaje 'BET', 4 de payload size, 2 de batch size, y 4 de batch id    

y payload := [APUESTA1];[APUESTA2];[APUESTA3];...   
donde cada apuesta tiene el mismo formato que cuando se mandaba una sola, definido en el Ej5.   

Ahora cada cliente tiene un volume de docker con su respectivo archivo csv en el sistema host, y tiene el valor batchsize configurable en client/config.yaml. El cliente lee de a batchsize entries del archivo, crea un array de apuestas y le delega el envío y recepción de confirmación al módulo de comunicación. Este se encarga de formatear el mensaje y colocarle su correspondiente header y de interpretar la respuesta del servidor.    

El servidor ahora interpreta los mensajes de tipo BET en su módulo de comunicación y devuelve un ACK al cliente si todas las apuestas de ese mensaje pudieron ser parseadas correctamente (y si estas son la misma cantidad que indicaba el header del mensaje que fueron enviadas).

## EJ 7:

El protocolo de comunicación fue extendido y ahora contiene 2 tipos más de mensajes, FIN y WIN. 

FIN es enviado al servidor por el cliente para notificar que está listo para recibir los resultados.   

El servidor almacena en su estado a los sockets de los clientes listos y, cuando todos lo están, les envía a cada uno WIN con los DNIs de los ganadores de su agencia.    

Cada vez que el servidor maneja una nueva conexión aceptada, solicita el header del mensaje entrante a communication.py con la función get_header() y según el tipo de mensaje que sea (BET o FIN) procede de manera pertinente. 

El cliente una vez que manda FIN espera recibir WIN de parte del servidor con los DNIs ganadores de su agencia y tras recibirlos, libera los recursos y concluye su ejecución.

## EJ 8:

### Consideraciones Generales

El servidor funciona en muchos aspectos como lo hacía antes, el thread principal acepta conexiones, y crea un nuevo thread para cada una que se recibe. Estos threads viven para el procesamiento del mensaje que se recibe y el envío de la respuesta al cliente. Cuando uno de estos threads recibe el mensaje FIN de parte del cliente, almacena el socket por el que el cliente esperará la respuesta en el estado del servidor y muere. El enésimo thread que reciba el mensaje FIN, siendo n la cantidad de agencias, realizará el sorteo y le enviará a los clientes sus ganadores. De esta manera se asegura el sincronismo del envío de los mensajes WIN a los clientes.

### Multithreading vs Multiprocessing

Considerando que el Global Interpreter Lock de Python no permite que varios hilos se ejecuten verdaderamente en paralelo en múltiples núcleos de CPU, reduciendo la eficiencia de las operaciones computacionales paralelizables, pense al principio en encarar el problema con multiprocessing.    

Por como fui programando el trabajo en puntos anteriores y la lógica que ya había hecho me resultó mucho más fácil implementar una solución multithread que una multiprocessing en la que probablemente haya tenido que definir un protocolo de mediana complejidad para IPC, o refactorizar básicamente todo. Para tomar la decisión final, también consideré que muchas de las operaciones que realiza el servidor son de lectura y escritura en sockets, que pueden implicar mucha espera según la conexión, y que aunque hubiera usado multiprocessing, como la persistencia de los datos no es process-safe, igualmente deberían esperar a que otros procesos liberen el storage para poder operar sobre él, es decir, hay cierto grado de concurrencia que justifica el uso de threads (que son más livianos que procesos) para resolver el problema.

### Exclusión Mutua

Para todos los recursos compartidos que precisa el manejo de conexión con el cliente, usé locks. Uno de los locks es para el acceso a almacenamiento mediante las funciones de utils.py y los demás son para el estado del servidor. Cuando un thread precisa usar uno de los recursos, pide el lock, y cuando finaliza su uso, lo libera.    

## Protocolo Versión Final 

El protocolo final del TP en funcionalidad es equivalente a lo mencionado en puntos anteriores, pero ahora el tamaño de los headers es mayor, los mensajes que envía el servidor al cliente (ACK, WIN) tienen un header fijo de 12 bytes y los mensajes que envía el cliente al servidor (BET, FIN) tienen un header fijo de 23 bytes.

### Mensaje de tipo ACK

ACK := [HEADER_ACK];[PAYLOAD_ACK]   
HEADER_ACK := ACK,[PAYLOAD_SIZE]    
PAYLOAD_ACK := [AGENCY_ID],[BATCH_ID]   

El servidor responde a los mensajes de tipo BET enviados por el cliente con un mensaje ACK, confirmando el batch de apuestas que envió el cliente. [PAYLOAD_SIZE] tiene un tamaño fijo de 8 bytes, por lo que [HEADER_ACK] tiene un tamaño fijo de 12 bytes (3 de 'ACK', 1 de ',' y 4 de [PAYLOAD_SIZE]).

### Mensaje de tipo WIN

WIN := [HEADER_WIN];[PAYLOAD_WIN]   
HEADER_WIN := WIN,[PAYLOAD_SIZE]    
PAYLOAD_WIN:= [DNI_1],[DNI_2],[DNI_3],...   

Una vez que el servidor recibe mensajes de tipo FIN de todas las agencias, les envía los DNIs de sus ganadores correspondientes empaquetado en un mensaje WIN. [PAYLOAD_SIZE] tiene un tamaño fijo de 8 bytes, por lo que [HEADER_WIN] tiene un tamaño fijo de 12 bytes.

### Mensaje de tipo BET

BET := [HEADER_BET];[PAYLOAD_BET]   
[HEADER_BET] := BET[PAYLOAD_SIZE][BATCH_SIZE][BATCH_ID] 
[PAYLOAD_BET] := [APUESTA_1],[APUESTA_2],[APUESTA_3],...    
[APUESTA] := [agencia],[betID],[nombre],[apellido],[DNI],[Fecha],[Numero]   

El cliente envía mensajes de tipo BET para registrar una tanda de apuestas. En su header incluye la cantidad de apuestas que envía [BATCHSIZE], el ID del batch para el que esperará recibir un ACK, y el tamaño del payload de las apuestas en sí. Para el header bet: [PAYLOAD_SIZE] tiene un tamaño fijo de 8 bytes, [BATCH_SIZE] de 4 bytes, y [BATCH_ID] de 8 bytes; por lo que el header tiene un tamaño fijo total de 23 bytes.

### Mensaje de tipo FIN

FIN := FIN[AGENCY_ID]

El cliente envía mensaje de tipo fin para indicar que ya envió todas las apuestas y que espera un mensaje de tipo WIN para recibir los DNIs de sus ganadores. [AGENCY_ID] tiene un tamaño fijo de 20 bytes, que implcia que FIN tiene un tamaño fijo de 23 bytes, esto es util porque el servidor al leer los primeros 23 bytes de un mensaje recibido, o ya tiene toda la información relevante para handlear un mensaje FIN, o procede con el manejo de un mensaje BET.   
Cuando el servidor recibe mensajes FIN de todas las agencias, revisa los ganadores y envía los mensajes WIN correspondientes.

### Observaciones Generales y mejoras

Si bien cuando se envían batchsizes grandes, de digamos 200 o 2000 apuestas se justifica un header de 23 bytes, para mensajes más cortos es bastante overhead. Si se quisiera mantener las mismas utilidades, (como mandar un ACK al batchID correspondiente ó revisar que efectivamente llegaron tantas bets como se indico en [BATCHSIZE]), pero con un header de menor tamaño, la mejor manera sería encodeando los datos numéricos de mejor manera, y reducir al mínimo delimitadores en donde se pueda, i.e.: ',' y ';'.