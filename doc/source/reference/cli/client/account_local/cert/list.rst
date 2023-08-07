===================================================
[OBSOLÈTE] Lister les certificats d'un compte local
===================================================

.. program:: waarp-gateway account local cert list

Affiche les informations des certificats du compte suivant les critères donnés.

.. option:: -l <LIMIT>, --limit=<LIMIT>

   Le nombre de maximum certificats autorisés dans la réponse. Fixé à 20 par défaut.

.. option:: -o <OFFSET>, --offset=<OFFSET>

   Le numéro du premier certificat renvoyé.

.. option:: -s <SORT>, --sort=<SORT>

   L'ordre et paramètre selon lequel les certificats seront triés. Les choix
   possibles sont:

   - tri par nom (``name+`` & ``name-``)

**Exemple**

.. code-block:: shell

   waarp-gateway -a 'http://user:password@localhost:8080' account local 'serveur_sftp' cert 'tata' list -l 10 -o 5 -s 'name-'
