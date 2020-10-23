=====================================
Ajouter un certificat à un partenaire
=====================================

.. program:: waarp-gateway partner cert add

.. describe:: waarp-gateway <ADDR> partner cert <PARTNER> add

Attache un nouveau certificat au partenaire donné à partir des informations renseignées.

.. option:: -n <NAME>, --name=<NAME>

   Le nom du certificat. Doit être unique pour le partenaire concerné.

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

   waarp-gateway http://user:password@localhost:8080 partner cert waarp_sftp add -n cert_waarp -p /waarp.pub -b /waarp.key -c /waarp.pem