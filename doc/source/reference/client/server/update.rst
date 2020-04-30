===================
Modifier un serveur
===================

.. program:: waarp-gateway server update

.. describe:: waarp-gateway <ADDR> server update <SERVER>

Remplace les attributs du serveur donné en paramètre par ceux fournis.
Les attributs omis resteront inchangés.

.. option:: -n <NAME>, --name=<NAME>

   Le nom du serveur. Doit être unique.

.. option:: -p <PROTO>, --protocol=<PROTO>

   Le protocole utilisé par le serveur.

.. option:: -r <ROOT>, --root=<ROOT>

   Le dossier racine du serveur.

.. option:: -c <CONF>, --config=<CONF>

   La configuration protocolaire du serveur.

|

**Exemple**

.. code-block:: bash

   waarp-gateway http://user:password@localhost:8080 server update serveur_sftp -n server_sftp_new -r /sftp/root_new -p sftp -c '{"address": "localhost", "port": 80}'
