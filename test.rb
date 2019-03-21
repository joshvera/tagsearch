# frozen_string_literal: true
require "ripper-tags/parser"
require "ctags"
require "elastomer/adapters/git_repository_code_iterator"

module GitHub
  module Jobs

    class TagsearchIndex < Job
      areas_of_responsibility :search

      def self.perform(*args)
        new(*args).perform
      end

      attr_reader :repo_id
      attr_reader :repo
      attr_reader :replica

      def initialize(repo_id, reindex = false)
        @repo_id = repo_id
        @repo = Repository.find(repo_id)
        @replica = @repo.replicas.first
      end

      def perform
        return unless GitHub.tagsearch_enabled?
        return unless repo

        restraint.lock!(repo_id, 1, 60.minutes) do
          head_sha = repo.refs[repo.default_branch].sha
          prev_sha = replica.rpc.tagsearch_read_commit_oid
          if prev_sha == head_sha
            return
          end

          code_iterator = Elastomer::Adapters::GitRepositoryCodeIterator.new repo,
            base_commit_sha: prev_sha

          code_iterator.each do |blob|
            if blob.deleted?
              replica.git_command('tagsearch-index', {:raise => true}, "-replace", old_path)
            else
              index_blob(blob)
            end
          end

          replica.rpc.tagsearch_write_commit_oid(head_sha)
        end
      rescue GitHub::Restraint::UnableToLock
      end

      def index_blob(blob)
        return if blob.path =~ %r|\Avendor/|

        case blob.language.try(:name)
        when "Ruby"
          tags = RipperTags::Parser.extract(blob.data, blob.path)
        when "C", "C++", "C#", "PHP", "Python", "Perl", "Javascript", "Java", "Pascal", "MatLab", "OCaml", "Lua", "Erlang", "Fortran", "Verilog", "Tcl", "Scheme", "Sh", "Go"
          tags = Ctags.tags_for_file(blob.path, blob.data)
        else
          return
        end

        tags_data = tags.map{ |t| t.to_json }.join("\n")
        replica.git_command('tagsearch-index', {:input => tags_data, :raise => true}, "-replace", blob.path)
      end

      def restraint
        @restraint ||= GitHub::Restraint.new GitHub.transient_redis
      end

      def self.queue
        :index_high
      end
    end

  end
end
