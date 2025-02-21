================================
Modifier une clé cryptographique
================================

.. program:: waarp-gateway key update

Remplace les attributs de la clé demandée avec ceux donnés. Les attributs
omis restent inchangés.

**Commande**

.. code-block:: shell

   waarp-gateway key update "<NAME>"

**Options**

.. option:: -n <NAME>, --name=<NAME>

   Le nom de la clé cryptographique. Doit être unique.

.. option:: -t <TYPE>, --type=<TYPE>

   Le type de clé cryptographique. Les valeurs acceptées sont :

   - ``AES`` pour une clé de chiffrement AES
   - ``HMAC`` pour une clé de signature HMAC
   - ``PGP-PUBLIC`` pour les clés PGP publiques
   - ``PGP-PRIVATE`` pour les clés PGP privées

.. option:: -k <KEY>, --key=<KEY>

   Le chemin vers le fichier contenant la clé. Celle-ci doit impérativement être
   en format textuel. Si la clé est en format binaire, celle-ci doit alors
   impérativement être convertie en format Base64 au préalable.

**Exemple**

.. code-block:: shell

   waarp-gateway user add "waarp-pgp-key" -n "waarp-pgp-pkey" -t "PGP-PRIVATE" -k "./waarp_pgp_pkey.pem"
