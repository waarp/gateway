=================
Ajouter un client
=================

.. program:: waarp-gateway client add

.. describe:: waarp-gateway client add

Ajoute un nouveau client avec les attributs renseignés.

.. option:: -n <NAME>, --name=<NAME>

   Le nom du client. Doit être unique.

.. option:: -p <PROTO>, --protocol=<PROTO>

   Le protocole utilisé par le client.

.. option:: -a <ADDRESS>, --local-address=<ADDRESS>

   L'adresse locale du client au format [adresse:port]. Cette option est
   facultative, par défaut, le client n'est donc pas restreint à une adresse
   particulière, et les choix de l'interface réseau et du port sont donc délégués
   au routeur de l'OS.

.. option:: -c <KEY:VAL>, --config=<KEY:VAL>

   La configuration protocolaire du client. Répéter pour chaque paramètre de la
   configuration. Les options de la configuration varient en fonction du protocole
   utilisé (voir :ref:`configuration protocolaire <reference-proto-config>` pour
   plus de détails).

|

**Exemple**

.. code-block:: shell

   waarp-gateway client add --name 'sftp_client' --protocol 'sftp' --local-address '192.168.1.2:8022' --config 'keyExchanges:["ecdh-sha2-nistp256"]'
