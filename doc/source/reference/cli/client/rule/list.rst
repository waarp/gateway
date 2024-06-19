.. _reference-cli-client-rules-list:

#################
Lister les règles
#################

.. program:: waarp-gateway rule list

Affiche une liste de toutes les règles remplissant les critères ci-dessous.

**Commande**

.. code-block:: shell

   waarp-gateway rule list

**Options**

.. option:: -l <LIMIT>, --limit=<LIMIT>

   Le nombre de maximum règles à afficher. Fixé à 20 par défaut.

.. option:: -o <OFFSET>, --offset=<OFFSET>

   Les `n` premières règles de la liste seront omises.

.. option:: -s <SORT>, --sort=<SORT>

   L'ordre et le paramètre de tri des règles. Les choix possibles sont :

   - tri par nom (``name+`` & ``name-``)

**Exemple**

.. code-block:: shell

   waarp-gateway rule list -l 10 -o 5 -s 'name-'
