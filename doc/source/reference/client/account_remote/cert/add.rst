=========================================
Ajouter un certificat à un compte distant
=========================================

.. program:: waarp-gateway account remote cert add

.. describe:: waarp-gateway <ADDR> account remote <PARTNER> cert <LOGIN> add

Attache un nouveau certificat au compte donné à partir des informations renseignées.

.. option:: -n <NAME>, --name=<NAME>

   Le nom du certificat. Doit être unique pour le compte concerné.

.. option:: -p <PRIV_KEY>, --private_key=<PRIV_KEY>

   Le chemin vers le fichier contenant la clé privée du certificat.

.. option:: -b <PUB_KEY>, --public_key=<PUB_KEY>

   Le chemin vers le fichier contenant la clé publique du certificat.

.. option:: -c <CERT>, --certificate=<CERT>

   Le chemin vers le fichier contenant le certificat du compte, avec
   la chaîne de certification complète.

|

**Exemple**

.. code-block:: shell

   waarp-gateway http://user:password@localhost:8080 account remote waarp_sftp cert titi add -n cert_titi -p /titi.pub -b /titi.key -c /titi.pem