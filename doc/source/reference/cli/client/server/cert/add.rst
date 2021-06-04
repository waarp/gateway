==================================
Ajouter un certificat à un serveur
==================================

.. program:: waarp-gateway server cert add

.. describe:: waarp-gateway <ADDR> server cert <SERVER> add

Attache un nouveau certificat au serveur donné à partir des informations renseignées.

.. option:: -n <NAME>, --name=<NAME>

   Le nom du certificat. Doit être unique pour le serveur concerné.

.. option:: -c <CERT>, --certificate=<CERT>

   Le chemin vers le fichier contenant le certificat du serveur, avec
   la chaîne de certification complète en format PEM. Mutuellement exclusif avec
   l'option `-b` (ou `--public_key`).

.. option:: -p <PRIV_KEY>, --private_key=<PRIV_KEY>

   Le chemin vers le fichier contenant la clé privée du certificat en format PEM.

.. option:: -b <PUB_KEY>, --public_key=<PUB_KEY>

   Le chemin vers le fichier contenant la clé publique SSH de l'entité en format
   *authorized_key*. Mutuellement exclusif avec l'option `-c` (ou `--certificate`).

|

**Exemple**

.. code-block:: shell

   waarp-gateway http://user:password@localhost:8080 server cert gw_r66 add -n cert_r66 -c /r66.crt -b /r66.key