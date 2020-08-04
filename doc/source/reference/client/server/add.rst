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

   Le dossier racine du serveur. Peut être un chemin relatif ou absolu. Si
   le chemin est relatif, il sera relatif à la racine de la *gateway* renseignée
   dans le fichier de configuration.

.. option:: -i <IN_DIR>, --in=<IN_DIR>

   Le dossier de réception du serveur. Peut être un chemin relatif ou absolu. Si
   le chemin est relatif, il sera relatif à la racine du serveur.

.. option:: -o <OUT_DIR>, --out=<OUT_DIR>

   Le dossier d'envoi du serveur. Peut être un chemin relatif ou absolu. Si
   le chemin est relatif, il sera relatif à la racine du serveur.

.. option:: -w <WORK_DIR>, --work=<WORK_DIR>

   Le dossier temporaire du serveur. Peut être un chemin relatif ou absolu. Si
   le chemin est relatif, il sera relatif à la racine du serveur.

.. option:: -c <CONF>, --config=<CONF>

   La configuration du serveur en format JSON. Contient les informations
   nécessaires pour lancer le serveur. Le contenu de la configuration
   varie en fonction du protocole utilisé, cette configuration est stockée en
   format JSON *raw*.

|

**Exemple**

.. code-block:: bash

   waarp-gateway http://user:password@localhost:8080 server add -n server_sftp -r /sftp/root  -p sftp -c '{"address":"localhost","port":21}'
