############################
Lister les identifiants SMTP
############################

.. program:: waarp-gateway email credential list

Affiche la liste des identifiants SMTP remplissant les critères ci-dessous.

**Commande**

.. code-block:: shell

   waarp-gateway email credential list

**Options**

.. option:: -l <LIMIT>, --limit=<LIMIT>

   Le nombre maximum d'identifiant autorisés dans la réponse. Fixé à 20 par défaut.

.. option:: -o <OFFSET>, --offset=<OFFSET>

   Le numéro du premier identifiant renvoyé.

.. option:: -s <SORT>, --sort=<SORT>

   Le paramètre et l'ordre selon lesquels les identifiants seront affichés. Les
   choix possibles sont:

   - par nom (``email+`` & ``email-``)

**Exemple**

.. code-block:: shell

   waarp-gateway email credential list -l "10" -o "5" -s "email+"
