==================================================
[OBSOLÈTE] Ajouter un certificat à un compte local
==================================================

.. program:: waarp-gateway account local cert add

.. describe:: waarp-gateway account local <SERVER> cert <LOGIN> add

Attache un nouveau certificat au compte donné à partir des informations renseignées.

.. option:: -n <NAME>, --name=<NAME>

   Le nom du certificat. Doit être unique pour le compte concerné.

.. option:: -c <CERT>, --certificate=<CERT>

   Le chemin vers le fichier contenant le certificat TLS du client, avec
   la chaîne de certification complète en format PEM. Mutuellement exclusif avec
   l'option `-b` (ou `--public_key`).

.. option:: -b <PUB_KEY>, --public_key=<PUB_KEY>

   Le chemin vers le fichier contenant la clé publique SSH du client en format
   *authorized_key*. Mutuellement exclusif avec l'option `-c` (ou `--certificate`).

|

**Exemple**

.. code-block:: shell

   waarp-gateway -a 'http://user:password@localhost:8080' account local 'serveur_sftp' cert 'tata' add -n 'key_tata' -b './tata.pub'