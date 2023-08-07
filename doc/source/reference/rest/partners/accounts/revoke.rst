.. _reference-rest-partners-accounts-revoke:

##########################################
Interdire à un compte d'utiliser une règle
##########################################

.. http:put:: /api/partners/(string:partner_name)/revoke/(string:rule)

   Retire au partenaire demandé la permission d'utiliser la règle donnée. Retire
   également la permission à tous les comptes rattachés à ce partenaire n'ayant pas
   été explicitement autorisés.

   :reqheader Authorization: Les identifiants de l'utilisateur

   :statuscode 200: La permission a été donnée avec succès
   :statuscode 401: Authentification d'utilisateur invalide
   :statuscode 404: Le partenaire ou la règle demandée n'existe pas


   **Exemple de requête**

   .. code-block:: http

      DELETE https://my_waarp_gateway.net/api/partners/waarp_sftp/authorize/rule_1 HTTP/1.1
      Authorization: Basic QWxhZGRpbjpvcGVuIHNlc2FtZQ==

   **Exemple de réponse**

   .. code-block:: http

      HTTP/1.1 200 OK
