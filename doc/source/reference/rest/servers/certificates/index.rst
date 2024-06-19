###############################################
[OBSOLÈTE] Gestion des certificats d'un serveur
###############################################

.. deprecated:: 0.9.0
   Les certificats sont désormais gérés via les *handlers* de gestion des
   :ref:`reference-auth-methods`. Les *handlers* suivants sont donc dépréciés.

Le point d'accès pour gérer les certificats d'un serveur local est
``/api/servers/{nom_serveur}/certificates``.

.. toctree::
   :maxdepth: 1

   create
   list
   consult
   update
   replace
   delete