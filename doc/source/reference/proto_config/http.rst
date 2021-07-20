Configuration HTTP
##################

L'objet JSON de configuration du protocole HTTP est identique pour les serveurs
et les partenaires. Il ne contient qu'une seule option :

* **useHTTPS** (*boolean*) - Indique si le partenaire/serveur utilise HTTPS au
  lieu de HTTP.

**Exemple**

.. code-block:: json

   {
     "useHTTPS": true
   }