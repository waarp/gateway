================================
Afficher une clé cryptographique
================================

.. program:: waarp-gateway key get

Affiche les informations de la clé donnée en paramètre. La valeur d'une clé
secrète (AES, HMAC, PGP privée) n'est jamais affichée : la commande indique
seulement qu'elle est définie (``<set>``). Une clé PGP, publique ou privée, est
décrite par ses identités et son empreinte.

**Commande**

.. code-block:: shell

   waarp-gateway key get "<NAME>"

**Options**

.. option:: --format=<FORMAT>

   Spécifie le format du retour de la commande. Les valeurs acceptées sont :
   ``human``, ``json`` et ``yaml``. Par défaut, le format sera le format pour
   humain (``human``).

|

**Exemple**

.. code-block:: shell

   waarp-gateway key get "waarp-pgp-key"
