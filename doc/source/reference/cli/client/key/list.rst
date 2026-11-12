================================
Lister les clés cryptographiques
================================

.. program:: waarp-gateway key list

Affiche une liste de toutes les clés cryptographiques remplissant les critères
ci-dessous. La valeur d'une clé secrète (AES, HMAC, PGP privée) n'est jamais
affichée : la commande indique seulement qu'elle est définie (``<set>``).

**Commande**

.. code-block:: shell

   waarp-gateway key list

**Options**

.. option:: -l <LIMIT>, --limit=<LIMIT>

   Le nombre maximum de clés autorisées dans la réponse. Fixé à 20 par défaut.

.. option:: -o <OFFSET>, --offset=<OFFSET>

   Le numéro de la première clé renvoyée.

.. option:: -s <SORT>, --sort=<SORT>

   Le paramètre et l'ordre selon lesquels les clés seront affichées. Les choix
   possibles sont :

   - par nom (``name+`` & ``name-``)
   - par type (``type+`` & ``type-``)

.. option:: --format=<FORMAT>

   Spécifie le format du retour de la commande. Les valeurs acceptées sont :
   ``human``, ``json`` et ``yaml``. Par défaut, le format sera le format pour
   humain (``human``).

|

**Exemple**

.. code-block:: shell

   waarp-gateway key list -l "10" -o "5" -s "name+"
