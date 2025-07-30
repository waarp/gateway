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

.. option:: --nb-of-attempts=<ATTEMPTS>

   Le nombre de fois que les transferts effectués avec ce client seront automatiquement
   retentés en cas d'échec.

.. option:: --first-retry-delay=<DELAY>

   Le délai entre la tentative de transfert initiale et la première reprise automatique.
   Les unités de temps acceptées sont : ``h`` (heures), ``m`` (minutes) et ``s`` (secondes).
   Plusieurs unités peuvent être combinées ensemble (ex: ``1h15m30s``).

.. option:: --retry-increment-factor=<FACTOR>

   Le facteur par lequel le délai décris ci-dessus sera multiplié entre chaque nouvelle
   tentative d'un transfert. Par exemple, si le délai initial est de 30s et que le
   facteur est de 2, alors le délai sera doublé à chaque nouvelle tentative (30s,
   puis 1m, 2m, 4m, etc) jusqu'à ce que le transfert réussisse ou bien que le nombre
   de tentatives soit épuisé.

|

**Exemple**

.. code-block:: shell

   waarp-gateway client update sftp_client --name 'sftp_client2' --protocol 'sftp' --local-address '192.168.1.2:8022' --config 'keyExchanges:["ecdh-sha2-nistp256"]'
