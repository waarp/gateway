####################
Lister les autorités
####################

.. program:: waarp-gateway authority list

.. describe:: waarp-gateway authority list <NAME>

Affiche une liste de toutes les autorités remplissant les critères ci-dessous.

.. option:: -l <LIMIT>, --limit=<LIMIT>

   Le nombre maximum d'autorités autorisées dans la réponse. Fixé à 20 par défaut.

.. option:: -o <OFFSET>, --offset=<OFFSET>

   Le numéro de la première autorité renvoyée.

.. option:: -s <SORT>, --sort=<SORT>

   Le paramètre et l'ordre selon lesquels les autorités seront affichées. Les
   choix possibles sont:

   - par nom (``name+`` & ``name-``)

|

**Exemple**

.. code-block:: shell

   waarp-gateway authority list -l 10 -o 5 -s 'name+'
