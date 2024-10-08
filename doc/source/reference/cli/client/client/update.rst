==================
Modifier un client
==================

.. program:: waarp-gateway client update

Remplace les attributs du client demandé avec ceux donnés. Les attributs omis
restent inchangés.

**Commande**

.. code-block:: shell

   waarp-gateway client update "<CLIENT>"

**Options**

.. option:: -n <NAME>, --name=<NAME>

   Le nom du client. Doit être unique.

.. option:: -p <PROTO>, --protocol=<PROTO>

   Le protocole utilisé par le client.

.. option:: -a <LOCAL-ADDRESS>, --local-address=<LOCAL-ADDRESS>

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

   waarp-gateway client update sftp_client --name 'sftp_client2' --protocol 'sftp' --local-address '192.168.1.2:8022' --config 'keyExchanges:["ecdh-sha2-nistp256"]'
