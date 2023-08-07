=====================================================
[OBSOLÈTE] Lister les certificats d'un compte distant
=====================================================

.. program:: waarp-gateway account remote cert list

Affiche les informations des certificats du compte suivant les critères donnés.

.. option:: -l <LIMIT>, --limit=<LIMIT>

   Le nombre de maximum certificats à afficher. Fixé à 20 par défaut.

.. option:: -o <OFFSET>, --offset=<OFFSET>

   Le numéro du premier certificat renvoyé.

.. option:: -s <SORT>, --sort=<SORT>

   L'ordre et paramètre selon lequel les certificats seront triés. Les choix
   possibles sont:

   - tri par nom (``name+`` & ``name-``)

**Exemple**

.. code-block:: shell

   waarp-gateway -a 'http://user:password@localhost:8080' account remote 'waarp_sftp' cert 'titi' list -l 10 -o 5 -s 'name-'
