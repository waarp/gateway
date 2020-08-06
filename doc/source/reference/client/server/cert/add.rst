==================================
Ajouter un certificat à un serveur
==================================

.. program:: waarp-gateway server cert add

.. describe:: waarp-gateway <ADDR> server cert <SERVER> add

Attache un nouveau certificat au serveur donné à partir des informations renseignées.

.. option:: -n <NAME>, --name=<NAME>

   Le nom du certificat. Doit être unique pour le serveur concerné.

.. option:: -p <PRIV_KEY>, --private_key=<PRIV_KEY>

   Le chemin vers le fichier contenant la clé privée du certificat.

.. option:: -b <PUB_KEY>, --public_key=<PUB_KEY>

   Le chemin vers le fichier contenant la clé publique du certificat.

.. option:: -c <CERT>, --certificate=<CERT>

   Le chemin vers le fichier contenant le certificat du serveur, avec
   la chaîne de certification complète.

|

**Exemple**

.. code-block:: shell

   waarp-gateway http://user:password@localhost:8080 server cert serveur_sftp add -n cert_sftp -p /sftp.pub -b /sftp.key -c /sftp.pem