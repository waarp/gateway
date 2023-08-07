======================
Modifier un partenaire
======================

.. program:: waarp-gateway partner update

Remplace les attributs du partenaire donné par ceux fournis ci-dessous. Les
attributs omis resteront inchangés.

.. option:: -n <NAME>, --name=<NAME>

   Le nom du partenaire. Doit être unique.

.. option:: -p <PROTO>, --protocol=<PROTO>

   Le protocole utilisé par le partenaire.

.. option:: -a <ADDRESS>, --address=<ADDRESS>

   L'adresse du partenaire (au format ``adresse:port``).

.. option:: -c <KEY:VAL>, --config=<KEY:VAL>

   La configuration protocolaire du partenaire. Répéter pour chaque paramètre de la
   configuration. Les options de la configuration varient en fonction du protocole
   utilisé (voir :ref:`configuration protocolaire <reference-proto-config>` pour
   plus de détails).

**Exemple**

.. code-block:: shell

   waarp-gateway -a 'http://user:password@localhost:8080' partner update 'partenaire_sftp' -n 'partner_sftp_new' -p 'sftp' -a 'waarp.fr:2022'
