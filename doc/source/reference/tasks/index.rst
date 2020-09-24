.. _tasks:

###########
Traitements
###########

.. todo:: 
   
   documenter les substitutions

Lors de l'ajout d'une règle, les traitements de la règle doivent être fournis
avec leurs arguments sous forme d'un objet JSON. Cet objet JSON contient 2
attributs:

* **type** (*string*) - Le type de traitement (voir liste ci-dessous).
* **args** (*object*) - Les arguments du traitement en format JSON. La structure
  de cet objet JSON dépend du type du traitement.

Cette rubrique documente les objets JSON contenant les arguments requis pour
l'ajout de chaque type de traitement.

**Exemple**

.. code-block:: json

   {
     "type": "COPY",
     "args": {
       "path": "/backup"
     }
   }

**Liste des traitements avec leurs arguments**

.. toctree::
   :maxdepth: 1

   copy
   copyrename
   delete
   exec
   execmove
   execoutput
   move
   moverename
   rename
   transfer
