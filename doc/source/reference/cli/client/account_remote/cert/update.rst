========================================
Modifier un certificat de compte distant
========================================

.. program:: waarp-gateway account remote cert update

.. describe:: waarp-gateway account remote <PARTNER> cert <LOGIN> update <CERT>

Change les attributs du certificat donné. Les noms du partenaire, du compte et du
certificat doivent être renseignés en arguments de programme. Les attributs omis
resteront inchangés.

.. option:: -n <NAME>, --name=<NAME>

   Le nom du certificat. Doit être unique pour un compte donné.

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

   waarp-gateway http://user:password@localhost:8080 account remote waarp_sftp cert titi update cert_titi -n cert_titi2 -p /titi2.pub -b titi2.key -c titi2.pem
