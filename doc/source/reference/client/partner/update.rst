======================
Modifier un partenaire
======================

.. program:: waarp-gateway partner update

.. describe:: waarp-gateway <ADDR> partner update <PARTNER>

Remplace les attributs du partenaire donné par ceux fournis ci-dessous. Les
attributs omis resteront inchangés.

.. option:: -n <NAME>, --name=<NAME>

   Le nom du partenaire. Doit être unique.

.. option:: -p <PROTO>, --protocol=<PROTO>

   Le protocole utilisé par le partenaire.

.. option:: -c <CONF>, --config=<CONF>

   La configuration protocolaire du partenaire en format JSON. Contient les
   informations nécessaires pour se connecter au partenaire. Le contenu de la
   configuration varie en fonction du protocole utilisé.

|

**Exemple**

.. code-block:: bash

   waarp-gateway http://user:password@localhost:8080 partner update partenaire_sftp -n partner_sftp_new -p sftp -c '{"address":"waarp.fr","port":80}'
