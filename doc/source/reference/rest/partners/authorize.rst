Autoriser un partenaire à utiliser une règle
============================================

.. http:put:: /api/partners/(string:partner_name)/authorize/(string:rule)

   Authorise le partenaire demandé à utiliser la règle donnée. Donner une permission
   à un partenaire donne automatiquement cette permission à tous les comptes rattachés
   à ce partenaire.

   :reqheader Authorization: Les identifiants de l'utilisateur

   :statuscode 200: La permission a été donnée avec succès
   :statuscode 401: Authentification d'utilisateur invalide
   :statuscode 404: Le partenaire ou la règle demandés n'existent pas


   .. admonition:: Exemple de requête

      .. code-block:: http

         DELETE https://my_waarp_gateway.net/api/partners/waarp_sftp/authorize/rule_1 HTTP/1.1
         Authorization: Basic QWxhZGRpbjpvcGVuIHNlc2FtZQ==

   .. admonition:: Exemple de réponse

      .. code-block:: http

         HTTP/1.1 200 OK