Supprimer une règle
===================

.. http:delete:: /api/rules/(int:rule_id)

   Supprime la règle portant le numéro ``rule_id``.

   **Requête**

   :reqheader Authorization: Les identifiants de l'utilisateur

   **Exemple de requête**

       .. code-block:: http

          DELETE https://my_waarp_gateway.net/api/rules/1 HTTP/1.1
          Authorization: Basic QWxhZGRpbjpvcGVuIHNlc2FtZQ==


   **Réponse**

   :statuscode 204: La règle a été supprimée avec succès
   :statuscode 401: Authentification d'utilisateur invalide
   :statuscode 404: La règle demandée n'existe pas

   **Exemple de réponse**

       .. code-block:: http

          HTTP/1.1 204 NO CONTENT