====================================
Modifier un certificat de partenaire
====================================

.. program:: waarp-gateway partner cert update

.. describe:: waarp-gateway <ADDR> partner cert <PARTNER> update <CERT>

Remplace les attributs du certificat par ceux renseignés ci-dessous. Les
attributs omis resteront inchangés.

.. option:: -n <NAME>, --name=<NAME>

   Le nom du certificat. Doit être unique pour un partenaire donné.

.. option:: -p <PRIV_KEY>, --private_key=<PRIV_KEY>

   Le chemin vers le fichier contenant la clé privée du certificat.

.. option:: -b <PUB_KEY>, --public_key=<PUB_KEY>

   Le chemin vers le fichier contenant la clé publique du certificat.

.. option:: -c <CERT>, --certificate=<CERT>

   Le chemin vers le fichier contenant le certificat du partenaire, avec
   la chaîne de certification complète.

|

**Exemple**

.. code-block:: shell

   waarp-gateway http://user:password@localhost:8080 partner cert waarp_sftp update cert_waarp -n cert_waarp2 -p /waarp2.pub -b waarp2.key -c waarp2.pem
