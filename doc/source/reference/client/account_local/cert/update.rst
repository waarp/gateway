======================================
Modifier un certificat de compte local
======================================

.. program:: waarp-gateway account local cert update

.. describe:: waarp-gateway <ADDR> account local <SERVER> cert <LOGIN> update <CERT>

Remplace les attributs du certificats par ceux renseignés ci-dessous.

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

.. code-block:: bash

   waarp-gateway http://user:password@localhost:8080 account local serveur_sftp cert tata update cert_tata -n cert_tata2 -p /tata2.pub -b tata2.key -c tata2.pem
