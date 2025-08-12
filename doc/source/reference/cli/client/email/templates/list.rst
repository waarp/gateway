###########################
Lister les template d'email
###########################

.. program:: waarp-gateway email template list

Affiche la liste des templates d'email remplissant les critères ci-dessous.

**Commande**

.. code-block:: shell

   waarp-gateway email template list

**Options**

.. option:: -l <LIMIT>, --limit=<LIMIT>

   Le nombre maximum de templates autorisés dans la réponse. Fixé à 20 par défaut.

.. option:: -o <OFFSET>, --offset=<OFFSET>

   Le numéro du premier template renvoyé.

.. option:: -s <SORT>, --sort=<SORT>

   Le paramètre et l'ordre selon lesquels les template seront affichés. Les
   choix possibles sont:

   - par nom (``name+`` & ``name-``)

**Exemple**

.. code-block:: shell

   waarp-gateway email template list -l "10" -o "5" -s "name+"
