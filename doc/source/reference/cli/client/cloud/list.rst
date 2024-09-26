==========================
Lister les instances cloud
==========================

.. program:: waarp-gateway cloud list

Affiche une liste des instances cloud connues.

**Commande**

.. code-block:: shell

   waarp-gateway cloud list

**Options**

.. option:: -l <LIMIT>, --limit=<LIMIT>

   Le nombre maximum d'instances autorisées dans la réponse. Fixé à 20 par défaut.

.. option:: -o <OFFSET>, --offset=<OFFSET>

   Le numéro de la première instance renvoyée.

.. option:: -s <SORT>, --sort=<SORT>

   Le paramètre et l'ordre selon lesquels les instances seront affichés. Les
   choix possibles sont:

   - par nom d'utilisateur (``name+`` & ``name-``)

**Exemple**

.. code-block:: shell

   waarp-gateway cloud list -l 10 -o 5 -s 'name+'
