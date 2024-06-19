######################################################
[OBSOLÈTE] Gestion des certificats d'un compte distant
######################################################

.. deprecated:: 0.9.0
   Les certificats sont désormais gérés via les *handlers* de gestion des
   :ref:`reference-auth-methods`. Les *handlers* suivants sont donc dépréciés.

Le point d'accès pour gérer les certificats d'un compte distant est
``/api/partners/{nom_partenaire}/accounts/{login}/certificates``.

.. toctree::
   :maxdepth: 1

   create
   list
   consult
   update
   replace
   delete