=================================
Modifier un certificat de serveur
=================================

.. program:: waarp-gateway server cert update

.. describe:: waarp-gateway <ADDR> server cert <SERVER> update <CERT>

Change les attributs du certificat donné. Les noms du serveur et du certificat
doivent être renseignés en arguments de programme. Les attributs omis resteront
inchangés.

.. option:: -n <NAME>, --name=<NAME>

   Le nom du certificat. Doit être unique pour un serveur donné.

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

   waarp-gateway http://user:password@localhost:8080 server cert serveur_sftp update cert_sftp -n cert_sftp2 -p /sftp2.pub -b sftp2.key -c sftp2.pem
