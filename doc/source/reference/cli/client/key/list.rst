==============================
Lister les clé cryptographique
==============================

.. program:: waarp-gateway key list

Affiche une liste de tous les clé cryptographiques remplissant les critères ci-dessous.

**Commande**

.. code-block:: shell

   waarp-gateway key list

**Options**

.. option:: -l <LIMIT>, --limit=<LIMIT>

   Le nombre maximum d'utilisateurs autorisés dans la réponse. Fixé à 20 par défaut.

.. option:: -o <OFFSET>, --offset=<OFFSET>

   Le numéro du premier utilisateur renvoyé.

.. option:: -s <SORT>, --sort=<SORT>

   Le paramètre et l'ordre selon lesquels les clés seront affichées. Les choix
   possibles sont :

   - par nom (``name+`` & ``name-``)
   - par type (``type+`` & ``type-``)

**Exemple**

.. code-block:: shell

   waarp-gateway key list -l "10" -o "5" -s "name+"
