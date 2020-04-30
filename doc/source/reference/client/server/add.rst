==================
Ajouter un serveur
==================

.. program:: waarp-gateway server add

.. describe:: waarp-gateway <ADDR> server add

Ajoute un nouveau serveur de transfert à la gateway avec les attributs fournis.

.. option:: -n <NAME>, --name=<NAME>

   Le nom du serveur. Doit être unique.

.. option:: -p <PROTO>, --protocol=<PROTO>

   Le protocole utilisé par le serveur.

.. option:: -r <ROOT>, --root=<ROOT>

   Le dossier racine du serveur.

.. option:: -c <CONF>, --config=<CONF>

   La configuration du serveur en format JSON. Contient les informations
   nécessaires pour lancer le serveur. Le contenu de la configuration
   varie en fonction du protocole utilisé, cette configuration est stockée en
   format JSON *raw*.

|

**Exemple**

.. code-block:: bash

   waarp-gateway http://user:password@localhost:8080 server add -n server_sftp -r /sftp/root -p sftp -c '{"address":"localhost","port":21}'
